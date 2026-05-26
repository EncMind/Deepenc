import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'app-terms',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="legal-container">
      <div class="legal-content">
        <header class="legal-header">
          <h1>Deepenc Terms of Service</h1>
          <p class="effective-date">Effective September 19, 2025</p>
        </header>

        <div class="legal-text">
          <p>Welcome to Deepenc! Before you access our services, please read these Terms of Service.</p>

          <p>These Terms of Service ("Terms") govern your use of Deepenc.com, Deepenc Plus, and other products and services that we may offer for individuals, along with any associated apps, software, and websites (together, our "Services"). These Terms are a contract between you and Deepenc ("Deepenc") and they include our Acceptable Use Policy. By accessing our Services, you agree to these Terms.</p>

          <p>Please read our Privacy Policy, which describes how we collect and use personal information.</p>

          <section>
            <h2>1. Who we are</h2>
            <p>Deepenc is an AI service platform providing secure, private, and universal access to multiple AI models including ChatGPT, Claude, and Gemini. We focus on user privacy and data security while delivering cutting-edge AI capabilities through our trusted execution environment (TEE) encryption technology.</p>
          </section>

          <section>
            <h2>2. Account creation and access</h2>
            <p><strong>Minimum age.</strong> You must be at least 18 years old or the minimum age required to consent to use the Services in your location, whichever is higher.</p>

            <p><strong>Your Deepenc Account.</strong> To access our Services, we may ask you to create an Account. You agree to provide correct, current, and complete Account information and allow us to use it to communicate with you about our Services. Our communications to you using your Account information will satisfy any requirements for legal notices.</p>

            <p>You may not share your Account login information or Account credentials with anyone else. You also may not make your Account available to anyone else. You are responsible for all activity occurring under your Account, and you agree to notify us immediately if you become aware of any unauthorized access to your Account by sending an email to support@deepenc.com.</p>

            <p>You may close your Account at any time by contacting us at support@deepenc.com.</p>

            <p><strong>Evaluation and Additional Services.</strong> In some cases, we may permit you to evaluate our Services for a limited time or with limited functionality. Use of our Services for evaluation purposes are for your personal, non-commercial use only.</p>

            <p>You may need to accept additional terms to use certain Services. These additional terms will supplement our Terms for those Services and may change your rights or obligations for those Services, including your obligations to pay fees.</p>
          </section>

          <section>
            <h2>3. Use of our Services</h2>
            <p>You may access and use our Services only in compliance with our Terms, including our Acceptable Use Policy, and any guidelines or supplemental terms we may post on the Services (the "Permitted Use"). You are responsible for all activity under the account through which you access the Services.</p>

            <p>You may not access or use, or help another person to access or use, our Services in the following ways:</p>
            <ul>
              <li>In any manner that violates any applicable law or regulation—including, without limitation, any laws about exporting data or software to and from the United States or other countries.</li>
              <li>To develop any products or services that compete with our Services, including to develop or train any artificial intelligence or machine learning algorithms or models or resell the Services.</li>
              <li>To decompile, reverse engineer, disassemble, or otherwise reduce our Services to human-readable form, except when these restrictions are prohibited by applicable law.</li>
              <li>To crawl, scrape, or otherwise harvest data or information from our Services other than as permitted under these Terms.</li>
              <li>To use our Services to obtain unauthorized access to any system or information, or to deceive any person.</li>
              <li>To infringe, misappropriate, or violate intellectual property or other legal rights (including the rights of publicity or privacy).</li>
              <li>To access the Services through automated or non-human means, whether through a bot, script, or otherwise, except where we explicitly permit it.</li>
              <li>To engage in any other conduct that restricts or inhibits any person from using or enjoying our Services, or that we reasonably believe exposes us—or any of our users, affiliates, or any other third party—to any liability, damages, or detriment of any type.</li>
              <li>To rely upon the Services to buy or sell securities or to provide or receive advice about securities, commodities, derivatives, or other financial products or services, as Deepenc is not a broker-dealer or a registered investment adviser.</li>
            </ul>

            <p>You also must not abuse, harm, interfere with, or disrupt our Services, including, for example, introducing viruses or malware, spamming or DDoSing Services, or bypassing any of our systems or protective measures.</p>
          </section>

          <section>
            <h2>4. Inputs, Outputs, and Materials</h2>
            <p><strong>Generally.</strong> You may interact with our Services by providing inputs ("Inputs"). Our Services may generate responses ("Outputs") based on your Inputs. Inputs and Outputs collectively are "Materials."</p>

            <p><strong>Rights and Responsibilities.</strong> You are responsible for all Inputs you submit to our Services. By submitting Inputs to our Services, you represent and warrant that you have all rights, licenses, and permissions necessary for us to process the Inputs under our Terms and to provide the Services to you. You also represent and warrant that your submitting Inputs to us will not violate our Terms, our Acceptable Use Policy, or any laws or regulations. As between you and Deepenc, and to the extent permitted by applicable law, you retain any right, title, and interest that you have in the Inputs you submit. Subject to your compliance with our Terms, we assign to you all of our right, title, and interest—if any—in Outputs.</p>

            <p><strong>Privacy and Data Protection.</strong> Unlike many AI services, Deepenc does NOT train on your data. All your conversations and data are encrypted using trusted execution environment (TEE) technology and remain private to you. We do not use your Materials for training AI models or improving our services unless you explicitly opt in.</p>

            <p><strong>Reliance on Outputs.</strong> Artificial intelligence and large language models are frontier technologies that are still improving in accuracy, reliability and safety. When you use our Services, you acknowledge and agree:</p>
            <ul>
              <li>Outputs may not always be accurate and may contain material inaccuracies even if they appear accurate because of their level of detail or specificity.</li>
              <li>You should not rely on any Outputs without independently confirming their accuracy.</li>
              <li>The Services and any Outputs may not reflect correct, current, or complete information.</li>
              <li>Outputs may contain content that is inconsistent with Deepenc's views.</li>
            </ul>
          </section>

          <section>
            <h2>5. Feedback</h2>
            <p>We appreciate feedback, including ideas and suggestions for improvement or rating an Output in response to an Input ("Feedback"). You have no obligation to give us Feedback, but if you do, you agree that we may use the Feedback however we choose without any obligation or other payment to you.</p>
          </section>

          <section>
            <h2>6. Subscriptions, fees and payment</h2>
            <p><strong>Fees and billing.</strong> You may be required to pay us fees to access or use our Services or certain features of our Services. You are responsible for paying any applicable fees listed for the Services unless otherwise communicated to you by Deepenc in writing.</p>

            <p>If you purchase access to our Services or features of our Services, you must provide complete and accurate billing information ("Payment Method"). You agree that we may charge the Payment Method for any applicable fees and any applicable tax. If the fees for these Services or features are specified to be recurring or based on usage, you agree that we may charge these fees and applicable taxes to the Payment Method on a periodic basis.</p>

            <p>Except as expressly provided in these Terms or where required by law, all payments are non-refundable. Please check your order carefully before confirming it.</p>

            <p><strong>Subscriptions.</strong> To access Deepenc Plus and other subscription services we may make available, you must sign up for a subscription with us (a "Subscription"), first by creating an Account, and then following the subscription procedure on our Services.</p>

            <p><strong>Subscription content, features, and services.</strong> The content, features, and other services provided as part of your Subscription, and the duration of your Subscription, will be described in the order process. We may change the content, features, and other services from time to time.</p>

            <p><strong>Subscription term and automatic renewal.</strong> If you sign up for a paid Subscription, we will automatically charge your Payment Method on each agreed-upon periodic renewal date until you cancel.</p>

            <p><strong>Subscription cancellation.</strong> You may cancel your Subscription for any reason by using a method we may provide to you through our products or by notifying us at support@deepenc.com. To avoid renewal and charges for the next term, cancel your subscription at least 24 hours before the renewal date.</p>
          </section>

          <section>
            <h2>7. Third-party services and links</h2>
            <p>Our Services may use or be used in connection with third-party content, services, or integrations. We do not control or accept responsibility for any loss or damage that may arise from your use of any third-party content, services, and integrations. Your use of any third-party content, services, and integrations is at your own risk and subject to any terms, conditions, or policies applicable to such third-party content, services, and integrations.</p>
          </section>

          <section>
            <h2>8. Content Moderation</h2>
            <p>If we become aware that any content (1) infringes another's copyright or any other intellectual property right, (2) is in breach of these Terms or our Acceptable Use Policy, or (3) may cause harm to Deepenc, our users, or third parties, we reserve the right to remove or take down some or all of such content.</p>
          </section>

          <section>
            <h2>9. Software</h2>
            <p>We may offer manual or automatic updates to our software including our apps ("Deepenc Software"), without advance notice to you.</p>
          </section>

          <section>
            <h2>10. Ownership of the Services</h2>
            <p>The Services are owned, operated, and provided by us and our affiliates, licensors, distributors, and service providers. We retain all of our respective rights, title, and interest, including intellectual property rights, in and to the Services. Other than the rights of access and use expressly granted in our Terms, our Terms do not grant you any right, title, or interest in or to our Services.</p>
          </section>

          <section>
            <h2>11. Disclaimer of warranties, limitations of liability, and indemnity</h2>
            <p>YOUR USE OF THE SERVICES AND MATERIALS IS SOLELY AT YOUR OWN RISK. THE SERVICES AND OUTPUTS ARE PROVIDED ON AN "AS IS" AND "AS AVAILABLE" BASIS AND, TO THE FULLEST EXTENT PERMISSIBLE UNDER APPLICABLE LAW, ARE PROVIDED WITHOUT WARRANTIES OF ANY KIND, WHETHER EXPRESS, IMPLIED, OR STATUTORY.</p>

            <p>TO THE FULLEST EXTENT PERMISSIBLE UNDER APPLICABLE LAW, IN NO EVENT WILL DEEPENC BE LIABLE FOR ANY DIRECT, INDIRECT, PUNITIVE, INCIDENTAL, SPECIAL, CONSEQUENTIAL, EXEMPLARY, OR OTHER DAMAGES ARISING OUT OF OR IN ANY WAY RELATED TO THE SERVICES OR THESE TERMS.</p>

            <p>TO THE FULLEST EXTENT PERMISSIBLE UNDER APPLICABLE LAW, DEEPENC'S TOTAL AGGREGATE LIABILITY TO YOU FOR ALL DAMAGES, LOSSES AND CAUSES OF ACTION WILL NOT EXCEED THE GREATER OF THE AMOUNT YOU PAID TO US FOR ACCESS TO OR USE OF THE SERVICES (IF ANY) IN THE SIX MONTHS PRECEDING THE DATE SUCH DAMAGES FIRST AROSE, AND $100.</p>
          </section>

          <section>
            <h2>12. General terms</h2>
            <p><strong>Changes to the Services.</strong> We may sometimes add or remove features, increase or decrease capacity limits, offer new Services, or stop offering certain Services. We reserve the right to modify, suspend, or discontinue the Services or your access to the Services, in whole or in part, at any time without notice to you.</p>

            <p><strong>Changes to these terms.</strong> We may revise and update these Terms at our discretion. If you continue to access the Services after we post the updated Terms, then you agree to the updated Terms. If you do not accept the updated Terms, you must stop using our Services.</p>

            <p><strong>Termination.</strong> You may stop accessing the Services at any time. We may suspend or terminate your access to the Services at any time without notice to you if we believe that you have breached these Terms. We may also terminate your Account if you have been inactive for over a year and you do not have a paid Account.</p>

            <p><strong>Governing law and exclusive jurisdiction.</strong> Our Terms will be governed by the laws of the State of California. You and Deepenc agree that any disputes arising out of or relating to these Terms will be resolved exclusively in the state or federal courts located in San Francisco, California.</p>
          </section>

          <section>
            <h2>Contact</h2>
            <p>If you have any questions about these Terms, please contact us at support@deepenc.com.</p>
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
export class TermsComponent {
}