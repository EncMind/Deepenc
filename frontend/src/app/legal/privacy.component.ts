import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'app-privacy',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="legal-container">
      <div class="legal-content">
        <header class="legal-header">
          <h1>Deepenc Acceptable Use Policy</h1>
          <p class="effective-date">Effective September 19, 2025</p>
        </header>

        <div class="legal-text">
          <p>This Acceptable Use Policy ("AUP") applies to anyone who can submit inputs to Deepenc's products and/or services, all of whom we refer to as "users." The AUP is intended to help our users stay safe and promote the responsible use of our products and services.</p>

          <p>The AUP is categorized according to who can use our products and for what purposes. We will update our policy as our technology and the associated risks evolve or as we learn about unanticipated risks.</p>

          <p><strong>Universal Usage Standards:</strong> Our Universal Usage Standards apply to all users and use cases.</p>
          <p><strong>High-Risk Use Case Requirements:</strong> Our High-Risk Use Case Requirements apply to specific consumer-facing use cases that pose an elevated risk of harm.</p>

          <p>Deepenc's Safety Team will implement detection and monitoring to enforce our AUP, so please review this policy carefully before using our products or services. If we learn that you have violated our AUP, we may throttle, suspend, or terminate your access to our products and services. We may also block or modify model outputs when inputs violate our AUP.</p>

          <p>If you believe that our model outputs are potentially inaccurate, biased or harmful, please notify us at safety@deepenc.com, or report it directly in our product through feedback features (where available).</p>

          <p>This AUP is calibrated to strike an optimal balance between enabling beneficial uses and mitigating potential harms.</p>

          <section>
            <h2>Universal Usage Standards</h2>

            <h3>Do Not Violate Applicable Laws or Engage in Illegal Activity</h3>

            <h3>Do Not Compromise Critical Infrastructure</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Facilitate the destruction or disruption of critical infrastructure such as power grids, water treatment facilities, medical devices, telecommunication networks, or air traffic control systems</li>
              <li>Obtain unauthorized access to critical systems such as voting machines, healthcare databases, and financial markets</li>
              <li>Interfere with the operation of military bases and related infrastructure</li>
            </ul>

            <h3>Do Not Compromise Computer or Network Systems</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Hack, breach, or otherwise gain unauthorized access to any computer, network, system, or account</li>
              <li>Distribute malware, viruses, worms, Trojan horses, or other harmful code</li>
              <li>Engage in any form of cyber attack, including denial-of-service attacks</li>
              <li>Bypass security measures or authentication systems</li>
            </ul>

            <h3>Do Not Develop or Design Weapons</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Design, create, or provide instructions for weapons, explosives, or other harmful devices</li>
              <li>Develop autonomous weapons systems</li>
              <li>Create chemical, biological, radiological, or nuclear weapons</li>
            </ul>

            <h3>Do Not Incite Violence or Hateful Behavior</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Promote, encourage, or incite violence against individuals or groups</li>
              <li>Generate content that promotes hatred, discrimination, or harassment based on race, ethnicity, religion, gender, sexual orientation, disability, or other protected characteristics</li>
              <li>Create content that threatens, intimidates, or bullies others</li>
            </ul>

            <h3>Do Not Compromise Privacy or Identity Rights</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Generate content that violates someone's privacy or impersonates others without consent</li>
              <li>Create deepfakes or other synthetic media designed to deceive</li>
              <li>Engage in doxing or sharing personal information without consent</li>
              <li>Facilitate identity theft or fraud</li>
            </ul>

            <h3>Do Not Compromise Children's Safety</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Generate content that sexualizes, grooms, or otherwise harms children</li>
              <li>Create content that could be used to exploit or abuse minors</li>
              <li>Facilitate illegal activities involving children</li>
            </ul>

            <h3>Do Not Create Psychologically or Emotionally Harmful Content</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Generate content that promotes self-harm, suicide, or eating disorders</li>
              <li>Create content designed to manipulate, deceive, or psychologically harm users</li>
              <li>Develop addictive or harmful applications targeting vulnerable populations</li>
            </ul>

            <h3>Do Not Create or Spread Misinformation</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Generate false or misleading information about important topics such as health, safety, or current events</li>
              <li>Create content designed to deceive people about factual matters</li>
              <li>Spread conspiracy theories or false information that could cause harm</li>
            </ul>

            <h3>Do Not Undermine Democratic Processes or Engage in Targeted Campaign Activities</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Generate content designed to suppress voting or mislead voters</li>
              <li>Create false information about electoral processes or candidates</li>
              <li>Engage in unauthorized political campaigning or lobbying</li>
            </ul>

            <h3>Do Not Use for Criminal Justice, Censorship, Surveillance, or Prohibited Law Enforcement Purposes</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Make decisions about criminal justice outcomes without proper human oversight</li>
              <li>Engage in mass surveillance of individuals</li>
              <li>Censor content or restrict access to information inappropriately</li>
            </ul>

            <h3>Do Not Engage in Fraudulent, Abusive, or Predatory Practices</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Commit fraud, including financial fraud or scams</li>
              <li>Generate content for phishing, spam, or other deceptive practices</li>
              <li>Engage in predatory behavior toward vulnerable individuals</li>
            </ul>

            <h3>Do Not Abuse our Platform</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Circumvent usage limits or access controls</li>
              <li>Reverse engineer or attempt to extract our models</li>
              <li>Use our services to train competing AI systems</li>
              <li>Engage in any activity that could damage, disable, or impair our services</li>
            </ul>

            <h3>Do Not Generate Sexually Explicit Content</h3>
            <p>This includes using our products or services to:</p>
            <ul>
              <li>Generate pornographic or sexually explicit content</li>
              <li>Create non-consensual intimate imagery</li>
              <li>Generate content that sexualizes individuals without consent</li>
            </ul>
          </section>

          <section>
            <h2>High-Risk Use Case Requirements</h2>

            <p>Some use cases pose an elevated risk of harm because they influence domains that are vital to public welfare and social equity. For these use cases, given potential risks to individuals and consumers, we believe that relevant human expertise should be integrated and that end-users should be aware when AI has been involved in producing outputs.</p>

            <p>As such, for the "High-Risk Use Cases" described below, we require that you implement these additional safety measures:</p>

            <p><strong>Human-in-the-loop:</strong> When using our products or services to provide advice, recommendations, or in subjective decision-making directly affecting individuals or consumers, a qualified professional in that field must review the content or decision prior to dissemination or finalization. You or your organization are responsible for the accuracy and appropriateness of that information.</p>

            <p><strong>Disclosure:</strong> If model outputs are presented directly to individuals or consumers, you must disclose to them that you are using AI to help produce your advice, decisions, or recommendations. This disclosure must be provided at a minimum at the beginning of each session.</p>

            <p>"High-Risk Use Cases" include:</p>
            <ul>
              <li><strong>Legal:</strong> Use cases related to legal interpretation, legal guidance, or decisions with legal implications</li>
              <li><strong>Healthcare:</strong> Use cases related to healthcare decisions, medical diagnosis, patient care, therapy, mental health, or other medical guidance. Wellness advice (e.g., advice on sleep, stress, nutrition, exercise, etc.) does not fall under this category</li>
              <li><strong>Insurance:</strong> Use cases related to health, life, property, disability, or other types of insurance underwriting, claims processing, or coverage decisions</li>
              <li><strong>Finance:</strong> Use cases related to financial decisions, including investment advice, loan approvals, and determining financial eligibility or creditworthiness</li>
              <li><strong>Employment and housing:</strong> Use cases related to decisions about the employability of individuals, resume screening, hiring tools, or other employment determinations or decisions regarding eligibility for housing, including leases and home loans</li>
              <li><strong>Academic testing, accreditation and admissions:</strong> Use cases related to standardized testing companies that administer school admissions, language proficiency, or professional certification exams</li>
              <li><strong>Media or professional journalistic content:</strong> Use cases related to using our products or services to automatically generate content and publish it for external consumption</li>
            </ul>
          </section>

          <section>
            <h2>Additional Guidelines</h2>
            <ul>
              <li>All consumer-facing chatbots must disclose to users that they are interacting with AI rather than a human. This disclosure must be provided at a minimum at the beginning of each chat session.</li>
              <li>Products serving minors must comply with additional safety guidelines and parental controls.</li>
            </ul>
          </section>

          <section>
            <h2>Reporting and Enforcement</h2>
            <p>If you encounter content or behavior that violates this AUP, please report it to safety@deepenc.com. We take all reports seriously and will investigate potential violations.</p>

            <p>Violations of this AUP may result in:</p>
            <ul>
              <li>Warning or educational outreach</li>
              <li>Temporary suspension of access</li>
              <li>Permanent termination of access</li>
              <li>Legal action where appropriate</li>
            </ul>
          </section>

          <section>
            <h2>Contact</h2>
            <p>If you have questions about this Acceptable Use Policy, please contact us at safety@deepenc.com.</p>
          </section>
        </div>

        <div class="back-link">
          <a routerLink="/auth" class="back-button">← Back to Sign In</a>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .legal-container {
      min-height: 100vh;
      background: #f8f9fa;
      padding: 2rem 1rem;
    }

    .legal-content {
      max-width: 800px;
      margin: 0 auto;
      background: white;
      border-radius: 12px;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
      padding: 3rem;
    }

    .legal-header {
      text-align: center;
      margin-bottom: 3rem;
      padding-bottom: 2rem;
      border-bottom: 2px solid #e1e5e9;
    }

    .legal-header h1 {
      margin: 0 0 1rem 0;
      color: #333;
      font-size: 2.5rem;
      font-weight: 700;
    }

    .effective-date {
      margin: 0;
      color: #666;
      font-size: 1.1rem;
      font-weight: 500;
    }

    .legal-text {
      line-height: 1.7;
      color: #444;
      font-size: 1rem;
    }

    .legal-text p {
      margin: 1.5rem 0;
    }

    .legal-text h2 {
      margin: 2.5rem 0 1.5rem 0;
      color: #333;
      font-size: 1.5rem;
      font-weight: 600;
      border-bottom: 1px solid #e1e5e9;
      padding-bottom: 0.5rem;
    }

    .legal-text h3 {
      margin: 2rem 0 1rem 0;
      color: #444;
      font-size: 1.25rem;
      font-weight: 600;
    }

    .legal-text section {
      margin: 2rem 0;
    }

    .legal-text ul {
      margin: 1rem 0 1rem 2rem;
    }

    .legal-text li {
      margin: 0.75rem 0;
    }

    .legal-text strong {
      color: #333;
      font-weight: 600;
    }

    .back-link {
      margin-top: 3rem;
      padding-top: 2rem;
      border-top: 1px solid #e1e5e9;
      text-align: center;
    }

    .back-button {
      display: inline-block;
      padding: 0.75rem 1.5rem;
      background: #667eea;
      color: white;
      text-decoration: none;
      border-radius: 8px;
      font-weight: 500;
      transition: background-color 0.2s ease;
    }

    .back-button:hover {
      background: #5a6fd8;
    }

    @media (max-width: 768px) {
      .legal-content {
        padding: 2rem 1.5rem;
      }

      .legal-header h1 {
        font-size: 2rem;
      }

      .legal-text {
        font-size: 0.95rem;
      }
    }
  `]
})
export class PrivacyComponent {
}